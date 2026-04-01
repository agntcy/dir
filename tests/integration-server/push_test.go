// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package integration_server

import (
	"encoding/json"
	"io"
	"sort"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	gormmodels "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/tests/test_utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gorm.io/gorm"
)

var _ = ginkgo.Describe("Push", func() {
	ginkgo.It("record", func(ctx ginkgo.SpecContext) {
		record := test_utils.FakeRecord()
		sort.Slice(record.Skills, func(i, j int) bool { return record.Skills[i].Id < record.Skills[j].Id })
		sort.Slice(record.Modules, func(i, j int) bool { return record.Modules[i].Name < record.Modules[j].Name })

		c := storev1.NewStoreServiceClient(t.conn)
		stream, err := c.Push(ctx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		err = stream.Send(&corev1.Record{Data: record.PbStruct()})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		err = stream.CloseSend()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		response, err := stream.Recv()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		cid := response.GetCid()
		gomega.Expect(cid).NotTo(gomega.BeEmpty())
		gomega.Expect(cid).NotTo(gomega.BeNil())

		// ORAS assertions
		manifestDesc, err := t.repository.Resolve(ctx, cid)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(manifestDesc.MediaType).To(gomega.Equal(ociv1.MediaTypeImageManifest))
		gomega.Expect(manifestDesc.Size).To(gomega.BeNumerically(">", 0))
		gomega.Expect(manifestDesc.Digest).To(gomega.HavePrefix("sha256:"))

		manifestReader, err := t.repository.Fetch(ctx, manifestDesc)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		defer manifestReader.Close()

		manifestData, err := io.ReadAll(manifestReader)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(manifestData).NotTo(gomega.BeEmpty())

		var manifest ociv1.Manifest

		err = json.Unmarshal(manifestData, &manifest)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(manifest.Versioned.SchemaVersion).To(gomega.Equal(2))
		gomega.Expect(manifest.MediaType).To(gomega.Equal(ociv1.MediaTypeImageManifest))
		gomega.Expect(manifest.ArtifactType).To(gomega.Equal(ociv1.MediaTypeImageManifest))
		gomega.Expect(manifest.Config.MediaType).To(gomega.Equal(ociv1.MediaTypeEmptyJSON))

		annotations := manifest.Annotations
		gomega.Expect(annotations).To(gomega.HaveKey("org.agntcy.dir/cid"))
		gomega.Expect(annotations).To(gomega.HaveKey("org.agntcy.dir/created-at"))
		gomega.Expect(annotations).To(gomega.HaveKey("org.agntcy.dir/name"))
		gomega.Expect(annotations).To(gomega.HaveKey("org.agntcy.dir/oasf-version"))
		gomega.Expect(annotations).To(gomega.HaveKey("org.agntcy.dir/schema-version"))
		gomega.Expect(annotations).To(gomega.HaveKey("org.agntcy.dir/type"))
		gomega.Expect(annotations).To(gomega.HaveKey("org.agntcy.dir/version"))
		gomega.Expect(annotations).To(gomega.HaveKey("org.opencontainers.image.created"))
		gomega.Expect(annotations["org.agntcy.dir/cid"]).To(gomega.Equal(cid))
		gomega.Expect(annotations["org.agntcy.dir/created-at"]).To(gomega.Equal(record.CreatedAt))
		gomega.Expect(annotations["org.agntcy.dir/name"]).To(gomega.Equal(record.Name))
		gomega.Expect(annotations["org.agntcy.dir/oasf-version"]).To(gomega.Equal("1.0.0"))
		gomega.Expect(annotations["org.agntcy.dir/schema-version"]).To(gomega.Equal(record.SchemaVersion))
		gomega.Expect(annotations["org.agntcy.dir/type"]).To(gomega.Equal("record"))
		gomega.Expect(annotations["org.agntcy.dir/version"]).To(gomega.Equal(record.Version))
		gomega.Expect(annotations["org.opencontainers.image.created"]).ToNot(gomega.BeEmpty())
		gomega.Expect(annotations["org.opencontainers.image.created"]).ToNot(gomega.BeNil())
		gomega.Expect(annotations["org.opencontainers.image.created"]).ToNot(gomega.Equal(record.CreatedAt))

		gomega.Expect(manifest.Layers).To(gomega.HaveLen(1))
		layerDesc := manifest.Layers[0]
		layerReader, err := t.repository.Fetch(ctx, layerDesc)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		defer layerReader.Close()

		layerData, err := io.ReadAll(layerReader)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(layerData).NotTo(gomega.BeEmpty())

		r, err := json.Marshal(record)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(layerData).To(gomega.MatchJSON(r))

		layerDigest, err := corev1.CalculateDigest(layerData)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(layerDigest).To(gomega.Equal(layerDesc.Digest))
		expectedCID, err := corev1.ConvertDigestToCID(layerDesc.Digest)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(cid).To(gomega.Equal(expectedCID))

		// GORM assertions
		recordModel, err := gorm.G[gormmodels.Record](t.db).Where("record_cid = ?", cid).Take(ctx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(recordModel.RecordCID).To(gomega.Equal(cid))
		gomega.Expect(recordModel.Name).To(gomega.Equal(record.Name))
		gomega.Expect(recordModel.SchemaVersion).To(gomega.Equal(record.SchemaVersion))
		gomega.Expect(recordModel.Version).To(gomega.Equal(record.Version))
		gomega.Expect(recordModel.OASFCreatedAt).To(gomega.Equal(record.CreatedAt))
		gomega.Expect(recordModel.Authors).To(gomega.Equal(record.Authors))
		gomega.Expect(recordModel.Signed).To(gomega.BeFalse())
		gomega.Expect(recordModel.CreatedAt).ToNot(gomega.BeNil())
		gomega.Expect(recordModel.UpdatedAt).ToNot(gomega.BeNil())

		skillModels, err := gorm.G[gormmodels.Skill](t.db).Where("record_cid = ?", cid).Order("skill_id").Find(ctx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(skillModels).To(gomega.HaveLen(len(record.Skills)))

		for idx, skillModel := range skillModels {
			skill := record.Skills[idx]
			gomega.Expect(skillModel.Name).To(gomega.Equal(skill.Name))
			gomega.Expect(skillModel.SkillID).To(gomega.Equal(uint64(skill.Id)))
			gomega.Expect(skillModel.RecordCID).To(gomega.Equal(cid))
			gomega.Expect(skillModel.CreatedAt).ToNot(gomega.BeNil())
			gomega.Expect(skillModel.UpdatedAt).ToNot(gomega.BeNil())
			gomega.Expect(skillModel.ID).To(gomega.BeNumerically(">", 0))
		}

		moduleModels, err := gorm.G[gormmodels.Module](t.db).Where("record_cid = ?", cid).Order("name").Find(ctx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(moduleModels).To(gomega.HaveLen(len(record.Modules)))

		for idx, moduleModel := range moduleModels {
			module := record.Modules[idx]
			gomega.Expect(moduleModel.Name).To(gomega.Equal(module.Name))
			gomega.Expect(moduleModel.RecordCID).To(gomega.Equal(cid))
			gomega.Expect(moduleModel.CreatedAt).ToNot(gomega.BeNil())
			gomega.Expect(moduleModel.UpdatedAt).ToNot(gomega.BeNil())
			gomega.Expect(moduleModel.ID).To(gomega.BeNumerically(">", 0))
		}

		locatorModels, err := gorm.G[gormmodels.Locator](t.db).Where("record_cid = ?", cid).Find(ctx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(locatorModels).To(gomega.HaveLen(len(record.Locators)))

		for idx, locatorModel := range locatorModels {
			locator := record.Locators[idx]

			gomega.Expect(locatorModel.RecordCID).To(gomega.Equal(cid))
			gomega.Expect(locatorModel.Type).To(gomega.Equal(locator.LocatorType))
			gomega.Expect(locatorModel.URL).To(gomega.Equal(locator.Urls[0]))
			gomega.Expect(locatorModel.CreatedAt).ToNot(gomega.BeNil())
			gomega.Expect(locatorModel.UpdatedAt).ToNot(gomega.BeNil())
			gomega.Expect(locatorModel.ID).To(gomega.BeNumerically(">", 0))
		}
	})
})
